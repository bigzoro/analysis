<template>
  <div class="concurrency-monitor">
    <h3>🔄 并发监控</h3>

    <div class="stats-grid">
      <!-- 智能工作者池统计 -->
      <div class="stat-card" v-if="stats.smart_worker_pool">
        <div class="stat-label">智能工作者池</div>
        <div class="stat-value">{{ stats.smart_worker_pool.WorkerCount }}</div>
        <div class="stat-details">
          处理: {{ stats.smart_worker_pool.TasksProcessed }} |
          队列: {{ stats.smart_worker_pool.QueueLength }}
        </div>
        <div class="stat-sub-details">
          平均处理时间: {{ (stats.smart_worker_pool.AvgProcessingTime / 1000000).toFixed(1) }}ms
        </div>
      </div>

      <!-- 缓存协程池统计 -->
      <div class="stat-card" v-if="stats.cache_pool">
        <div class="stat-label">缓存协程池</div>
        <div class="stat-value">{{ stats.cache_pool.running }}</div>
        <div class="stat-details">运行中 / 最大: {{ stats.cache_pool.max_workers }}</div>
      </div>

      <!-- 系统资源使用 -->
      <div class="stat-card" v-if="stats.system_resources">
        <div class="stat-label">Goroutines</div>
        <div class="stat-value">{{ stats.system_resources.goroutines }}</div>
        <div class="stat-details">活跃协程数量</div>
      </div>

      <!-- 内存使用 -->
      <div class="stat-card" v-if="stats.system_resources">
        <div class="stat-label">内存使用</div>
        <div class="stat-value">{{ formatBytes(stats.system_resources.memory_used) }}</div>
        <div class="stat-details">堆内存占用</div>
      </div>

      <!-- GC统计 -->
      <div class="stat-card" v-if="stats.system_resources">
        <div class="stat-label">GC周期</div>
        <div class="stat-value">{{ stats.system_resources.gc_cycles }}</div>
        <div class="stat-details">上次GC: {{ formatTimeAgo(stats.system_resources.last_gc) }}</div>
      </div>

      <!-- 熔断器状态 -->
      <div class="stat-card" v-if="circuitBreakers && circuitBreakers.length > 0">
        <div class="stat-label">熔断器</div>
        <div class="stat-value">{{ healthyCircuitBreakers }}/{{ circuitBreakers.length }}</div>
        <div class="stat-details">健康 / 总数</div>
      </div>
    </div>

    <!-- 熔断器详情 -->
    <div v-if="circuitBreakers && circuitBreakers.length > 0" class="circuit-breakers-section">
      <h4>🔌 熔断器状态</h4>
      <div class="circuit-breaker-list">
        <div
          v-for="cb in circuitBreakers"
          :key="cb.name"
          class="circuit-breaker-item"
          :class="getCircuitBreakerClass(cb)"
        >
          <div class="cb-name">{{ cb.name }}</div>
          <div class="cb-state">{{ getCircuitBreakerStateText(cb.state) }}</div>
          <div class="cb-stats">
            失败率: {{ cb.failure_rate.toFixed(1) }}% |
            请求: {{ cb.total_requests }}
          </div>
        </div>
      </div>
    </div>

    <!-- 资源池状态 -->
    <div v-if="resourcePools && Object.keys(resourcePools).length > 0" class="resource-pools-section">
      <h4>🏊 资源池状态</h4>
      <div class="resource-pool-list">
        <div
          v-for="(pool, type) in resourcePools"
          :key="type"
          class="resource-pool-item"
          :class="{ 'pool-healthy': pool.ActiveCount < pool.MaxSize * 0.8 }"
        >
          <div class="pool-name">{{ getResourceTypeName(type) }}</div>
          <div class="pool-stats">
            活跃: {{ pool.ActiveCount }} |
            空闲: {{ pool.IdleCount }} |
            总数: {{ pool.TotalCount }}
          </div>
        </div>
      </div>
    </div>

    <!-- 健康状态指示器 -->
    <div class="health-indicator">
      <div class="health-status" :class="overallHealthClass">
        <span class="health-icon">{{ overallHealthIcon }}</span>
        <span class="health-text">{{ overallHealthText }}</span>
      </div>
    </div>

    <div class="actions">
      <button @click="refreshStats" :disabled="loading" class="btn-primary">
        {{ loading ? '刷新中...' : '刷新统计' }}
      </button>
      <button @click="resetCircuitBreakers" :disabled="resetting" class="btn-warning">
        {{ resetting ? '重置中...' : '重置熔断器' }}
      </button>
      <button @click="scaleWorkerPool" :disabled="scaling" class="btn-secondary">
        {{ scaling ? '调整中...' : '调整工作者池' }}
      </button>
    </div>

    <div v-if="message" class="message" :class="{ 'message-success': message.type === 'success', 'message-error': message.type === 'error' }">
      {{ message.text }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api/api.js'

const stats = ref({})
const loading = ref(false)
const resetting = ref(false)
const scaling = ref(false)
const message = ref(null)

// 计算属性
const circuitBreakers = computed(() => {
  if (!stats.value.circuit_breakers) return []
  return Object.entries(stats.value.circuit_breakers).map(([name, stat]) => ({
    name,
    ...stat
  }))
})

const healthyCircuitBreakers = computed(() => {
  return circuitBreakers.value.filter(cb => cb.healthy).length
})

const resourcePools = computed(() => {
  return stats.value.resource_pools || {}
})

const overallHealth = computed(() => {
  // 计算整体健康度
  let healthScore = 100

  // 检查熔断器健康
  if (circuitBreakers.value.length > 0) {
    const healthyRatio = healthyCircuitBreakers.value / circuitBreakers.value.length
    if (healthyRatio < 0.8) healthScore -= 30
  }

  // 检查资源池健康
  for (const pool of Object.values(resourcePools.value)) {
    if (pool.ActiveCount > pool.MaxSize * 0.9) {
      healthScore -= 20
      break
    }
  }

  // 检查工作者池健康
  if (stats.value.smart_worker_pool) {
    const queueUtilization = stats.value.smart_worker_pool.QueueLength /
      (stats.value.smart_worker_pool.QueueLength + 100) // 假设队列容量为100
    if (queueUtilization > 0.8) healthScore -= 20
  }

  return Math.max(0, Math.min(100, healthScore))
})

const overallHealthClass = computed(() => {
  const health = overallHealth.value
  if (health >= 80) return 'health-good'
  if (health >= 60) return 'health-warning'
  return 'health-critical'
})

const overallHealthIcon = computed(() => {
  const health = overallHealth.value
  if (health >= 80) return '🟢'
  if (health >= 60) return '🟡'
  return '🔴'
})

const overallHealthText = computed(() => {
  const health = overallHealth.value
  if (health >= 80) return '系统健康'
  if (health >= 60) return '需要关注'
  return '系统异常'
})

onMounted(() => {
  refreshStats()
})

async function refreshStats() {
  loading.value = true
  try {
    const response = await api.get('/api/concurrency/stats')
    stats.value = response.data
    showMessage('统计信息已更新', 'success')
  } catch (error) {
    console.error('获取并发统计失败:', error)
    showMessage('获取统计信息失败', 'error')
  } finally {
    loading.value = false
  }
}

async function resetCircuitBreakers() {
  resetting.value = true
  try {
    await api.post('/api/circuit-breakers/reset')
    showMessage('熔断器已重置', 'success')
    // 重置后刷新统计
    setTimeout(refreshStats, 1000)
  } catch (error) {
    console.error('重置熔断器失败:', error)
    showMessage('重置熔断器失败', 'error')
  } finally {
    resetting.value = false
  }
}

async function scaleWorkerPool() {
  const newMin = prompt('输入最小工作者数量:', '2')
  const newMax = prompt('输入最大工作者数量:', '10')

  if (!newMin || !newMax) return

  scaling.value = true
  try {
    await api.post('/api/worker-pool/scale', {
      min_workers: parseInt(newMin),
      max_workers: parseInt(newMax)
    })
    showMessage('工作者池调整请求已发送', 'success')
  } catch (error) {
    console.error('调整工作者池失败:', error)
    showMessage('调整工作者池失败', 'error')
  } finally {
    scaling.value = false
  }
}

function getCircuitBreakerStateText(state) {
  switch (state) {
    case 0: return '关闭'
    case 1: return '开启'
    case 2: return '半开启'
    default: return '未知'
  }
}

function getCircuitBreakerClass(cb) {
  return {
    'cb-closed': cb.state === 0,
    'cb-open': cb.state === 1,
    'cb-half-open': cb.state === 2,
    'cb-unhealthy': !cb.healthy
  }
}

function getResourceTypeName(type) {
  const names = {
    0: '数据库',
    1: 'Redis',
    2: 'HTTP客户端',
    3: '工作者池',
    4: '内存',
    5: '文件句柄'
  }
  return names[type] || `类型${type}`
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatTimeAgo(timestamp) {
  if (!timestamp) return '未知'
  const now = new Date()
  const time = new Date(timestamp)
  const diffMs = now - time
  const diffSec = Math.floor(diffMs / 1000)

  if (diffSec < 60) return `${diffSec}秒前`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}分钟前`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}小时前`
  return `${Math.floor(diffSec / 86400)}天前`
}

function showMessage(text, type) {
  message.value = { text, type }
  setTimeout(() => {
    message.value = null
  }, 3000)
}
</script>

<style scoped>
.concurrency-monitor {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 16px;
  text-align: center;
  border-left: 4px solid #007bff;
}

.stat-card.pool-healthy {
  border-left-color: #28a745;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
}

.stat-details {
  font-size: 12px;
  color: #888;
  margin-bottom: 4px;
}

.stat-sub-details {
  font-size: 11px;
  color: #999;
}

.circuit-breakers-section,
.resource-pools-section {
  margin-bottom: 24px;
}

.circuit-breakers-section h4,
.resource-pools-section h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 16px;
}

.circuit-breaker-list,
.resource-pool-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.circuit-breaker-item,
.resource-pool-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-radius: 6px;
  background: #f8f9fa;
  border-left: 4px solid #6c757d;
}

.circuit-breaker-item.cb-closed {
  border-left-color: #28a745;
  background: #d4edda;
}

.circuit-breaker-item.cb-open {
  border-left-color: #dc3545;
  background: #f8d7da;
}

.circuit-breaker-item.cb-half-open {
  border-left-color: #ffc107;
  background: #fff3cd;
}

.cb-name {
  font-weight: 500;
  color: #333;
}

.cb-state {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 12px;
  background: #e9ecef;
  color: #495057;
}

.cb-stats {
  font-size: 12px;
  color: #666;
}

.pool-name {
  font-weight: 500;
  color: #333;
}

.pool-stats {
  font-size: 12px;
  color: #666;
}

.health-indicator {
  display: flex;
  justify-content: center;
  margin-bottom: 20px;
}

.health-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  border-radius: 20px;
  font-weight: 500;
}

.health-status.health-good {
  background: #d4edda;
  color: #155724;
}

.health-status.health-warning {
  background: #fff3cd;
  color: #856404;
}

.health-status.health-critical {
  background: #f8d7da;
  color: #721c24;
}

.health-icon {
  font-size: 18px;
}

.actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.btn-primary, .btn-secondary, .btn-warning {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background: #545b62;
}

.btn-warning {
  background: #ffc107;
  color: #212529;
}

.btn-warning:hover:not(:disabled) {
  background: #e0a800;
}

.btn-primary:disabled, .btn-secondary:disabled, .btn-warning:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.message {
  padding: 12px;
  border-radius: 4px;
  font-weight: 500;
}

.message-success {
  background: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.message-error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}
</style>











