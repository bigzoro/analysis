<template>
  <section class="panel">
    <div class="row topbar">
      <h2>运行记录</h2>
      <div class="spacer"></div>
      <label>交易所：</label>
      <select v-model="entity" @change="load" class="select">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>
      <div class="gap"></div>
      <input
        class="search"
        v-model.trim="keyword"
        placeholder="搜索 RunID…"
        @keyup.enter="handleSearch"
      />
      <button class="btn" @click="handleSearch">搜索</button>
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
    </div>

    <!-- 当前激活的筛选条件显示 -->
    <div v-if="hasActiveFilters" class="active-filters">
      <span class="filter-label">当前筛选：</span>
      <span v-if="keyword" class="filter-tag">
        关键词: {{ keyword }}
        <button class="tag-close" @click="keyword = ''; handleSearch()">×</button>
      </span>
      <span v-if="filters.startDate || filters.endDate" class="filter-tag">
        日期: {{ filters.startDate || '开始' }} ~ {{ filters.endDate || '结束' }}
        <button class="tag-close" @click="filters.startDate = ''; filters.endDate = ''; handleFilterChange('date')">×</button>
      </span>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel">
    <table class="tbl">
      <thead>
      <tr>
        <th>运行 ID</th>
        <th>交易所</th>
        <th>统计时点（UTC）</th>
        <th>入库时间</th>
        <th>资产总额（USD）</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="r in runs" :key="r.run_id">
        <td style="font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px;">{{ r.run_id }}</td>
        <td>{{ r.entity }}</td>
        <td>{{ new Date(r.as_of).toLocaleString() }}</td>
        <td>{{ new Date(r.created_at).toLocaleString() }}</td>
        <td>{{ r.total_usd }}</td>
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
      @change="onPaginationChange"
    />
  </section>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'
import { handleError } from '../utils/errorHandler.js'

const entities = ref([])
const entity = ref('binance')
const runs = ref([])
const page = ref(1)
const pageSize = ref(50)
const total = ref(0)
const totalPages = ref(1)
const loading = ref(false)

// 搜索和过滤
const keyword = ref('')
const showAdvancedFilters = ref(false)
const filters = ref({
  startDate: '',
  endDate: ''
})

const hasActiveFilters = computed(() => {
  return !!(keyword.value || filters.value.startDate || filters.value.endDate)
})

function handleSearch() {
  page.value = 1
  load()
}

function handleFilterChange(type) {
  page.value = 1
  load()
}

function clearFilters() {
  keyword.value = ''
  filters.value = {
    startDate: '',
    endDate: ''
  }
  page.value = 1
  load()
}

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes('binance') && entities.value.length) entity.value = entities.value[0]
}

async function load() {
  loading.value = true
  try {
    const params = {
      entity: entity.value,
      page: page.value,
      page_size: pageSize.value
    }
    if (keyword.value) {
      params.keyword = keyword.value
    }
    if (filters.value.startDate) {
      params.start_date = filters.value.startDate
    }
    if (filters.value.endDate) {
      params.end_date = filters.value.endDate
    }
    const r = await api.listRuns(params)
    runs.value = (r.items || r.runs || []).map(x => ({ ...x, total_usd: Number.parseFloat(x.total_usd || '0').toLocaleString() }))
    total.value = r.total || 0
    totalPages.value = r.total_pages || 1
    page.value = r.page || page.value
  } catch (e) {
    handleError(e, '加载运行记录', { showToast: false })
  } finally {
    loading.value = false
  }
}

function onPaginationChange({ page: newPage, pageSize: newPageSize }) {
  page.value = newPage
  pageSize.value = newPageSize
  load()
}

watch(entity, () => {
  page.value = 1
  load()
})

onMounted(async () => { await initEntities(); await load() })
</script>

<style scoped lang="scss">
.tbl { width:100%; border-collapse:collapse;
  th,td{ padding:10px; border-bottom:1px solid var(--border); }
  th{ color:var(--muted); text-align:left; }
  tbody tr:hover{ background:#121722; }
}

/* 搜索和过滤样式 */
.topbar {
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.gap {
  width: 12px;
}

.search {
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
  min-width: 200px;
}

.btn {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  cursor: pointer;
  font-size: 14px;
}

.btn:hover {
  background: #f3f4f6;
}

.btn-secondary {
  background: #f9fafb;
  border-color: #d1d5db;
}

.btn-clear {
  background: #fee2e2;
  border-color: #fca5a5;
  color: #991b1b;
}

.btn-clear:hover {
  background: #fecaca;
}

.select {
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
}

.advanced-filters {
  margin-top: 12px;
  padding: 12px;
  background: #f9fafb;
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-row label {
  min-width: 80px;
  font-size: 14px;
  color: var(--text);
}

.date-input {
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: #fff;
  color: var(--text);
  font-size: 14px;
  flex: 1;
  max-width: 200px;
}

.date-separator {
  color: var(--muted);
  font-size: 14px;
  margin: 0 4px;
}

.active-filters {
  margin-top: 12px;
  padding: 8px 12px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.filter-label {
  font-size: 14px;
  color: #1e40af;
  font-weight: 500;
}

.filter-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #dbeafe;
  border: 1px solid #93c5fd;
  border-radius: 6px;
  font-size: 13px;
  color: #1e3a8a;
}

.tag-close {
  background: none;
  border: none;
  color: #1e3a8a;
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.tag-close:hover {
  background: #bfdbfe;
}
</style>
