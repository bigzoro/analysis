<template>
  <section class="panel">
    <div class="row">
      <h2>回测分析</h2>
      <div class="spacer"></div>
      <button class="primary" @click="load">刷新</button>
    </div>
  </section>

  <!-- 统计概览 -->
  <section style="margin-top:12px;" class="panel" v-if="stats">
    <h3>回测统计</h3>
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">总记录数</div>
        <div class="stat-value">{{ stats.total || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">已完成</div>
        <div class="stat-value">{{ stats.completed || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">24h平均收益</div>
        <div class="stat-value" :class="getPerformanceClass(stats.avg_performance_24h)">
          {{ formatPercent(stats.avg_performance_24h) }}
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">7天平均收益</div>
        <div class="stat-value" :class="getPerformanceClass(stats.avg_performance_7d)">
          {{ formatPercent(stats.avg_performance_7d) }}
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">30天平均收益</div>
        <div class="stat-value" :class="getPerformanceClass(stats.avg_performance_30d)">
          {{ formatPercent(stats.avg_performance_30d) }}
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">24h胜率</div>
        <div class="stat-value">{{ formatPercent(stats.win_rate_24h) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">7天胜率</div>
        <div class="stat-value">{{ formatPercent(stats.win_rate_7d) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">30天胜率</div>
        <div class="stat-value">{{ formatPercent(stats.win_rate_30d) }}</div>
      </div>
    </div>
  </section>

  <!-- 回测记录列表 -->
  <section style="margin-top:12px;" class="panel">
    <h3>回测记录</h3>
    <div v-if="loading" style="text-align:center; padding: 40px;">
      <p>加载中...</p>
    </div>
    <div v-else-if="records.length === 0" style="text-align:center; padding: 40px;">
      <p>暂无回测记录</p>
    </div>
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>币种</th>
          <th>推荐时间</th>
          <th>推荐价格</th>
          <th>24h后价格</th>
          <th>24h收益</th>
          <th>7天后价格</th>
          <th>7天收益</th>
          <th>30天后价格</th>
          <th>30天收益</th>
          <th>状态</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="record in records" :key="record.id">
          <td><strong>{{ record.base_symbol }}</strong><br><small>{{ record.symbol }}</small></td>
          <td>{{ formatTime(record.recommended_at) }}</td>
          <td>{{ formatPrice(record.recommended_price) }}</td>
          <td>{{ formatPrice(record.price_after_24h) }}</td>
          <td :class="getPerformanceClass(record.performance_24h)">
            {{ formatPercent(record.performance_24h) }}
          </td>
          <td>{{ formatPrice(record.price_after_7d) }}</td>
          <td :class="getPerformanceClass(record.performance_7d)">
            {{ formatPercent(record.performance_7d) }}
          </td>
          <td>{{ formatPrice(record.price_after_30d) }}</td>
          <td :class="getPerformanceClass(record.performance_30d)">
            {{ formatPercent(record.performance_30d) }}
          </td>
          <td>
            <span :class="['status-badge', `status-${record.status}`]">
              {{ getStatusText(record.status) }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api/api.js'

const stats = ref(null)
const records = ref([])
const loading = ref(false)

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
    failed: '失败'
  }
  return map[status] || status
}

async function load() {
  loading.value = true
  try {
    const [statsRes, recordsRes] = await Promise.all([
      api.getBacktestStats(),
      api.getBacktestRecords({ limit: 50 })
    ])
    stats.value = statsRes
    records.value = recordsRes.records || []
  } catch (error) {
    console.error('加载回测数据失败:', error)
    alert('加载失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
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
</style>

