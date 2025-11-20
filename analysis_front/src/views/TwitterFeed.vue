<template>
  <div class="page">
    <section class="panel">
      <div class="row topbar">
        <h2>Twitter 用户帖子</h2>
        <div class="spacer"></div>

        <label>@用户：</label>
        <input
            v-model.trim="username"
            class="input"
            placeholder="例如: binance"
            @keyup.enter="fetchNow"
            @blur="onUsernameBlur"
        />
        <span v-if="username" class="username-display">@{{ username.replace(/^@/, '') }}</span>
        <label>每页：</label>
        <select v-model.number="pageSize" class="select">
          <option :value="5">5</option>
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
        </select>
        <label class="chk">
          <input type="checkbox" v-model="store" />
          入库
        </label>

        <button class="btn" @click="fetchNow" :disabled="loading || !username">
          {{ loading ? '拉取中…' : '获取' }}
        </button>
        <button class="btn" @click="loadLocal" :disabled="!username">
          查本地
        </button>
        <button class="btn btn-secondary" @click="showAdvancedFilters = !showAdvancedFilters">
          {{ showAdvancedFilters ? '收起' : '高级筛选' }}
        </button>
        <button 
          v-if="hasActiveFilters" 
          class="btn btn-clear" 
          @click="clearFilters"
          title="清除所有筛选"
        >
          清除筛选
        </button>
      </div>

      <!-- 高级筛选面板 -->
      <div v-if="showAdvancedFilters" class="advanced-filters">
        <div class="filter-row">
          <label>日期范围：</label>
          <input 
            type="date" 
            v-model="filters.startDate" 
            @change="handleFilterChange('date')"
            class="date-input"
          />
          <span class="date-separator">至</span>
          <input 
            type="date" 
            v-model="filters.endDate" 
            @change="handleFilterChange('date')"
            class="date-input"
          />
        </div>
        
        <div class="filter-row">
          <label>关键词搜索：</label>
          <input 
            type="text" 
            v-model.trim="filters.keyword" 
            @keyup.enter="handleFilterChange('keyword')"
            @change="handleFilterChange('keyword')"
            class="search-input"
            placeholder="搜索推文内容…"
          />
        </div>
      </div>

      <!-- 当前激活的筛选条件显示 -->
      <div v-if="hasActiveFilters" class="active-filters">
        <span class="filter-label">当前筛选：</span>
        <span v-if="filters.keyword" class="filter-tag">
          关键词: {{ filters.keyword }}
          <button class="tag-close" @click="filters.keyword = ''; handleFilterChange('keyword')">×</button>
        </span>
        <span v-if="filters.startDate || filters.endDate" class="filter-tag">
          日期: {{ filters.startDate || '开始' }} ~ {{ filters.endDate || '结束' }}
          <button class="tag-close" @click="filters.startDate = ''; filters.endDate = ''; handleFilterChange('date')">×</button>
        </span>
      </div>
    </section>

    <section class="panel" style="margin-top:12px;">
      <div v-if="loading" class="loading">正在从 X 拉取…</div>

      <template v-else>
        <div v-if="error" class="error">{{ error }}</div>
        <div v-else-if="list.length === 0" class="empty">暂无数据</div>

        <div v-else>
          <div class="info-bar">
            <span class="info-text">共 {{ total }} 条推文</span>
            <span class="info-text">第 {{ page }} / {{ totalPages }} 页</span>
            <span class="info-text" v-if="store">（已保存到数据库）</span>
          </div>
          <table class="table">
            <thead>
            <tr>
              <th class="col-time">时间</th>
              <th>内容</th>
              <th class="col-link">链接</th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="t in list" :key="t.tweet_id || t.id">
              <td class="mono">{{ fmtTime(t.tweet_time) }}</td>
              <td class="text">
                <div class="tweet-content" v-html="formatTweetText(t.text)"></div>
              </td>
              <td>
                <a :href="t.url" target="_blank" rel="noopener" class="link" :title="t.url">
                  打开 ↗
                </a>
              </td>
            </tr>
            </tbody>
          </table>
          <!-- 统一分页组件 -->
          <Pagination
            v-if="total > 0"
            v-model:page="page"
            v-model:pageSize="pageSize"
            :total="total"
            :totalPages="totalPages"
            :loading="loading"
            :pageSizeOptions="[5, 10, 20, 50, 100]"
            @change="onPaginationChange"
          />
        </div>
      </template>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'
import { handleError } from '../utils/errorHandler.js'

const username = ref('')
const pageSize = ref(5) // 每页显示数量
const page = ref(1) // 当前页码
const total = ref(0) // 总记录数
const totalPages = ref(1) // 总页数
const store = ref(true)
const loading = ref(false)
const list = ref([])

// 搜索和过滤
const showAdvancedFilters = ref(false)
const filters = ref({
  startDate: '',
  endDate: '',
  keyword: ''
})

const hasActiveFilters = computed(() => {
  return !!(filters.value.keyword || filters.value.startDate || filters.value.endDate)
})

function handleFilterChange(type) {
  page.value = 1
  loadLocal()
}

function clearFilters() {
  filters.value = {
    startDate: '',
    endDate: '',
    keyword: ''
  }
  page.value = 1
  loadLocal()
}
const error = ref('')
const nextToken = ref('')
const hasMore = ref(false)
const loadingMore = ref(false)

function fmtTime(iso) {
  try {
    const d = new Date(iso)
    const now = new Date()
    const diff = now - d
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)
    
    if (minutes < 1) return '刚刚'
    if (minutes < 60) return `${minutes}分钟前`
    if (hours < 24) return `${hours}小时前`
    if (days < 7) return `${days}天前`
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } catch { 
    return iso 
  }
}

function formatTweetText(text) {
  if (!text) return ''
  // 高亮链接、@提及、#话题
  return text
    .replace(/(https?:\/\/[^\s]+)/g, '<a href="$1" target="_blank" class="tweet-link">$1</a>')
    .replace(/@(\w+)/g, '<span class="tweet-mention">@$1</span>')
    .replace(/#(\w+)/g, '<span class="tweet-hashtag">#$1</span>')
}

async function fetchNow() {
  if (!username.value) return
  loading.value = true
  error.value = ''
  nextToken.value = ''
  hasMore.value = false
  page.value = 1 // 重置到第一页
  try {
    const data = await api.twitterFetch({
      username: username.value.replace(/^@/, ''),
      limit: pageSize.value,
      store: store.value ? 1 : 0,
    })
    // 处理新的响应格式（支持分页）
    if (data.items && Array.isArray(data.items)) {
      list.value = data.items
      nextToken.value = data.next_token || ''
      hasMore.value = data.has_more || false
    } else if (Array.isArray(data)) {
      // 兼容旧格式
      list.value = data
      hasMore.value = false
    } else {
      list.value = []
    }
    
    if (list.value.length === 0) {
      error.value = '未获取到推文，请检查用户名是否正确'
    }
  } catch (err) {
    error.value = err.message || '获取推文失败'
    list.value = []
    nextToken.value = ''
    hasMore.value = false
    handleError(err, '获取推文', { showToast: false })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (!nextToken.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const data = await api.twitterFetch({
      username: username.value.replace(/^@/, ''),
      limit: pageSize.value,
      store: store.value ? 1 : 0,
      pagination_token: nextToken.value,
    })
    
    if (data.items && Array.isArray(data.items)) {
      // 追加到现有列表
      list.value = [...list.value, ...data.items]
      nextToken.value = data.next_token || ''
      hasMore.value = data.has_more || false
    }
  } catch (err) {
    error.value = err.message || '加载更多失败'
    handleError(err, '加载更多', { showToast: false })
  } finally {
    loadingMore.value = false
  }
}

async function loadLocal() {
  loading.value = true
  error.value = ''
  try {
    const params = { 
      page: page.value,
      page_size: pageSize.value
    }
    if (username.value) {
      params.username = username.value.replace(/^@/, '')
    }
    if (filters.value.keyword) {
      params.keyword = filters.value.keyword
    }
    if (filters.value.startDate) {
      params.start_date = filters.value.startDate
    }
    if (filters.value.endDate) {
      params.end_date = filters.value.endDate
    }
    const data = await api.twitterPosts(params)
    
    // 处理分页响应格式
    if (data.items && Array.isArray(data.items)) {
      list.value = data.items
      total.value = data.total || 0
      totalPages.value = data.total_pages || 1
      page.value = data.page || 1
    } else if (Array.isArray(data)) {
      // 兼容旧格式（无分页）
      list.value = data
      total.value = data.length
      totalPages.value = 1
    } else {
      list.value = []
      total.value = 0
      totalPages.value = 1
    }
    
    if (list.value.length === 0) {
      if (username.value) {
        error.value = '本地数据库中没有该用户的推文记录'
      } else {
        error.value = '本地数据库中没有推文记录'
      }
    } else {
      error.value = '' // 清除错误信息
    }
  } catch (err) {
    error.value = err.message || '查询本地数据失败'
    list.value = []
    total.value = 0
    totalPages.value = 1
    handleError(err, '查询本地数据', { showToast: false })
  } finally {
    loading.value = false
  }
}

// 分页变化处理
function onPaginationChange({ page: newPage, pageSize: newPageSize }) {
  page.value = newPage
  pageSize.value = newPageSize
  loadLocal()
}

// 用户名输入框失去焦点时，如果有用户名就加载本地数据（重置到第一页）
function onUsernameBlur() {
  if (username.value) {
    page.value = 1
    loadLocal()
  }
}

// 移除 watch，由 Pagination 组件的 @change 事件统一处理

// 页面加载时自动加载本地数据（所有用户的最近推文）
onMounted(() => {
  loadLocal()
})
</script>

<style scoped>
.page { max-width: 1100px; margin: 0 auto; padding: 18px; }
.panel { border: 1px solid var(--border); border-radius: 12px; background: var(--panel); padding: 14px; color: var(--text); }
.row { display: flex; align-items: center; gap: 10px; }
.topbar h2 { margin: 0; font-size: 18px; font-weight: 600; }
.spacer { flex: 1; }
.input {
  height: 32px; width: 200px; border: 1px solid var(--border);
  background: #ffffff; color: var(--text); border-radius: 8px; padding: 0 10px;
}
.select {
  height: 32px; border: 1px solid var(--border);
  background: #ffffff; color: var(--text); border-radius: 8px; padding: 0 10px;
}
.chk { display: inline-flex; align-items: center; gap: 6px; color: var(--muted); }
.btn {
  height: 32px; padding: 0 12px; border: 1px solid var(--border);
  border-radius: 8px; background: #f3f4f6; color: #111827; cursor: pointer;
}
.btn:disabled { opacity: .6; cursor: default; }
.loading, .empty, .error { padding: 20px; text-align: center; }
.empty { color: var(--muted); }
.error { color: #ef4444; background: rgba(239, 68, 68, 0.1); border-radius: 8px; margin-bottom: 12px; }
.username-display { 
  font-size: 14px; 
  color: #1d9bf0; 
  font-weight: 500;
  padding: 4px 8px;
  background: rgba(29, 155, 240, 0.1);
  border-radius: 6px;
}
.info-bar { padding: 10px 12px; background: rgba(59, 130, 246, 0.1); border-radius: 8px; margin-bottom: 12px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.info-text { font-size: 14px; color: #3b82f6; }
.load-more-bar { padding: 20px; text-align: center; }
.load-more-bar .btn { min-width: 120px; }

.table { width: 100%; border-collapse: collapse; }
th, td { padding: 10px 12px; border-bottom: 1px dashed rgba(17,24,39,.06); vertical-align: top; }
thead th { color: #374151; font-weight: 500; }
.col-time { width: 200px; }
.col-link { width: 90px; text-align: center; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace; font-variant-numeric: tabular-nums; }
.text { white-space: pre-wrap; line-height: 1.6; }
.tweet-content { word-break: break-word; }
.tweet-link { color: #2563eb; text-decoration: underline; }
.tweet-mention { color: #1d9bf0; font-weight: 500; }
.tweet-hashtag { color: #1d9bf0; }
.link { color: #2563eb; text-decoration: none; font-weight: 500; }
.link:hover { text-decoration: underline; }

/* 搜索和过滤样式 */
.topbar {
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.gap {
  width: 12px;
}

.btn-secondary {
  background: #f3f4f6;
  color: #374151;
  border-color: rgba(0, 0, 0, 0.12);
}

.btn-secondary:hover {
  background: #e5e7eb;
}

.btn-clear {
  background: #fee2e2;
  color: #dc2626;
  border-color: rgba(220, 38, 38, 0.3);
}

.btn-clear:hover {
  background: #fecaca;
}

.advanced-filters {
  margin-top: 16px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-row label {
  min-width: 100px;
  font-size: 14px;
  color: var(--text);
  font-weight: 500;
}

.date-input, .search-input {
  height: 36px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
  flex: 1;
  max-width: 200px;
}

.search-input {
  max-width: 400px;
}

.date-separator {
  color: var(--muted);
  font-size: 14px;
  margin: 0 4px;
}

.active-filters {
  margin-top: 12px;
  padding: 12px;
  background: rgba(59, 130, 246, 0.05);
  border-radius: 8px;
  border: 1px solid rgba(59, 130, 246, 0.2);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-label {
  font-size: 14px;
  color: #3b82f6;
  font-weight: 500;
}

.filter-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #fff;
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 16px;
  font-size: 13px;
  color: #3b82f6;
}

.tag-close {
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 50%;
  color: #3b82f6;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.tag-close:hover {
  background: rgba(59, 130, 246, 0.2);
}
</style>
