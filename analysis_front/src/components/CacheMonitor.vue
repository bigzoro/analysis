<template>
  <div class="cache-monitor">
    <h3>缓存监控</h3>

    <div class="stats-grid">
      <!-- 后端缓存统计 -->
      <div class="stat-card">
        <div class="stat-label">后端缓存命中率</div>
        <div class="stat-value">{{ cacheStats.hit_rate ? (cacheStats.hit_rate * 100).toFixed(1) : 0 }}%</div>
        <div class="stat-details">
          命中: {{ cacheStats.hit_count || 0 }} | 未命中: {{ cacheStats.miss_count || 0 }}
        </div>
      </div>

      <!-- 前端缓存统计 -->
      <div class="stat-card">
        <div class="stat-label">前端缓存命中率</div>
        <div class="stat-value">{{ cacheStats.frontend?.overall?.hitRate || 0 }}%</div>
        <div class="stat-details">
          命中: {{ cacheStats.frontend?.overall?.hits || 0 }} | 未命中: {{ cacheStats.frontend?.overall?.misses || 0 }}
        </div>
      </div>

      <!-- Redis状态 -->
      <div class="stat-card">
        <div class="stat-label">Redis状态</div>
        <div class="stat-value" :class="{ 'status-enabled': cacheStats.redis_enabled, 'status-disabled': !cacheStats.redis_enabled }">
          {{ cacheStats.redis_enabled ? '已启用' : '未启用' }}
        </div>
        <div class="stat-details">分布式缓存</div>
      </div>

      <!-- 缓存健康度 -->
      <div class="stat-card">
        <div class="stat-label">缓存健康度</div>
        <div class="stat-value" :class="getHealthClass(cacheStats.healthScore)">
          {{ cacheStats.healthScore || 0 }}/100
        </div>
        <div class="stat-details">综合评分</div>
      </div>

      <!-- 内存缓存使用率 -->
      <div class="stat-card">
        <div class="stat-label">内存缓存</div>
        <div class="stat-value">{{ cacheStats.frontend?.layers?.memory?.utilization || 0 }}%</div>
        <div class="stat-details">
          {{ cacheStats.frontend?.layers?.memory?.total || 0 }}/{{ cacheStats.frontend?.layers?.memory?.maxSize || 100 }}
        </div>
      </div>

      <!-- 本地存储使用率 -->
      <div class="stat-card">
        <div class="stat-label">本地存储</div>
        <div class="stat-value">{{ cacheStats.frontend?.layers?.local?.utilization || 0 }}%</div>
        <div class="stat-details">
          {{ cacheStats.frontend?.layers?.local?.total || 0 }}/{{ cacheStats.frontend?.layers?.local?.maxSize || 1000 }}
        </div>
      </div>

      <!-- 预热服务状态 -->
      <div class="stat-card">
        <div class="stat-label">预热服务</div>
        <div class="stat-value" :class="{ 'status-enabled': !cacheStats.warmupStatus?.isRunning, 'status-disabled': cacheStats.warmupStatus?.isRunning }">
          {{ cacheStats.warmupStatus?.isRunning ? '运行中' : '空闲' }}
        </div>
        <div class="stat-details">{{ cacheStats.warmupStatus?.taskCount || 0 }} 个任务</div>
      </div>
    </div>

    <!-- 健康建议 -->
    <div v-if="cacheStats.recommendations && cacheStats.recommendations.length > 0" class="recommendations">
      <h4>💡 优化建议</h4>
      <div class="recommendation-list">
        <div v-for="rec in cacheStats.recommendations" :key="rec.action" class="recommendation-item" :class="`rec-${rec.type}`">
          <span class="rec-icon">{{ getRecommendationIcon(rec.type) }}</span>
          <span class="rec-text">{{ rec.message }}</span>
        </div>
      </div>
    </div>

    <div class="actions">
      <button @click="refreshStats" :disabled="loading" class="btn-primary">
        {{ loading ? '刷新中...' : '刷新统计' }}
      </button>
      <button @click="warmupCache" :disabled="warmingUp" class="btn-secondary">
        {{ warmingUp ? '预热中...' : '缓存预热' }}
      </button>
      <button @click="clearCache" :disabled="clearing" class="btn-danger">
        {{ clearing ? '清理中...' : '清理缓存' }}
      </button>
    </div>

    <div v-if="message" class="message" :class="{ 'message-success': message.type === 'success', 'message-error': message.type === 'error' }">
      {{ message.text }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '@/api/api.js'
import { getCacheStats, cacheMonitor, cacheWarmupService } from '@/utils/dataCache.js'

const cacheStats = ref({})
const loading = ref(false)
const warmingUp = ref(false)
const clearing = ref(false)
const message = ref(null)
const monitorData = ref(null)

// 监控更新监听器
let monitorListener = null

onMounted(() => {
  refreshStats()

  // 启动缓存监控
  cacheMonitor.startMonitoring()

  // 监听监控数据更新
  monitorListener = (data) => {
    monitorData.value = data
    updateCombinedStats()
  }
  cacheMonitor.addListener(monitorListener)
})

onUnmounted(() => {
  // 停止监控
  cacheMonitor.stopMonitoring()

  // 移除监听器
  if (monitorListener) {
    cacheMonitor.removeListener(monitorListener)
  }
})

// 更新组合统计信息
function updateCombinedStats() {
  if (!monitorData.value) return

  cacheStats.value = {
    ...cacheStats.value,
    ...monitorData.value,
    // 添加前端缓存统计
    frontend: getCacheStats()
  }
}

async function refreshStats() {
  loading.value = true
  try {
    // 获取后端缓存统计
    const response = await api.get('/api/cache/stats')
    cacheStats.value = {
      ...response.data,
      frontend: getCacheStats(),
      warmupStatus: cacheWarmupService.getStatus()
    }

    // 更新监控数据
    updateCombinedStats()

    showMessage('统计信息已更新', 'success')
  } catch (error) {
    console.error('获取缓存统计失败:', error)
    // 只显示前端统计
    cacheStats.value = {
      frontend: getCacheStats(),
      warmupStatus: cacheWarmupService.getStatus(),
      backend_error: true
    }
    showMessage('后端统计获取失败，仅显示前端统计', 'error')
  } finally {
    loading.value = false
  }
}

async function warmupCache() {
  warmingUp.value = true
  try {
    // 前端预热
    await cacheWarmupService.warmup()

    // 后端预热
    try {
      await api.post('/api/cache/warmup')
    } catch (e) {
      console.warn('后端预热失败:', e)
    }

    showMessage('缓存预热已完成', 'success')
    // 预热完成后刷新统计
    setTimeout(refreshStats, 1000)
  } catch (error) {
    console.error('缓存预热失败:', error)
    showMessage('缓存预热失败', 'error')
  } finally {
    warmingUp.value = false
  }
}

async function clearCache() {
  if (!confirm('确定要清理所有缓存吗？')) {
    return
  }

  clearing.value = true
  try {
    // 前端清理
    const { clearCache: clearFrontendCache } = await import('@/utils/dataCache.js')
    await clearFrontendCache()

    // 后端清理
    try {
      await api.post('/api/cache/clear')
    } catch (e) {
      console.warn('后端清理失败:', e)
    }

    showMessage('缓存已清理', 'success')
    // 清理完成后刷新统计
    setTimeout(refreshStats, 500)
  } catch (error) {
    console.error('清理缓存失败:', error)
    showMessage('清理缓存失败', 'error')
  } finally {
    clearing.value = false
  }
}

function showMessage(text, type) {
  message.value = { text, type }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

function getHealthClass(score) {
  if (!score) return 'status-disabled'
  if (score >= 80) return 'status-enabled'
  if (score >= 60) return 'status-warning'
  return 'status-error'
}

function getRecommendationIcon(type) {
  switch (type) {
    case 'error': return '🚨'
    case 'warning': return '⚠️'
    case 'info': return 'ℹ️'
    default: return '💡'
  }
}
</script>

<style scoped>
.cache-monitor {
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
}

.status-enabled {
  color: #28a745;
}

.status-disabled {
  color: #dc3545;
}

.actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.btn-primary, .btn-secondary, .btn-danger {
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

.btn-danger {
  background: #dc3545;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #c82333;
}

.btn-primary:disabled, .btn-secondary:disabled, .btn-danger:disabled {
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

.recommendations {
  margin-top: 24px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border-left: 4px solid #17a2b8;
}

.recommendations h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 16px;
}

.recommendation-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.recommendation-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.4;
}

.recommendation-item.rec-error {
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.recommendation-item.rec-warning {
  background: #fff3cd;
  color: #856404;
  border: 1px solid #ffeaa7;
}

.recommendation-item.rec-info {
  background: #d1ecf1;
  color: #0c5460;
  border: 1px solid #bee5eb;
}

.rec-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.rec-text {
  flex: 1;
}

.status-warning {
  color: #856404;
}

.status-error {
  color: #721c24;
}
</style>
