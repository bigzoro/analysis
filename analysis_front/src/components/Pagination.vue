<template>
  <div class="pagination" v-if="total > 0">
    <div class="pagination-info">
      <span>共 {{ total }} 条</span>
      <span>第 {{ page }} / {{ totalPages }} 页</span>
    </div>
    
    <div class="pagination-controls">
      <button 
        class="btn-page" 
        :disabled="page <= 1 || loading" 
        @click="goToPage(1)"
        title="首页"
      >
        ««
      </button>
      <button 
        class="btn-page" 
        :disabled="page <= 1 || loading" 
        @click="goToPage(page - 1)"
        title="上一页"
      >
        «
      </button>
      
      <span class="page-numbers">
        <button
          v-for="p in visiblePages"
          :key="p"
          class="btn-page"
          :class="{ active: p === page }"
          :disabled="loading"
          @click="goToPage(p)"
        >
          {{ p }}
        </button>
      </span>
      
      <button 
        class="btn-page" 
        :disabled="page >= totalPages || loading" 
        @click="goToPage(page + 1)"
        title="下一页"
      >
        »
      </button>
      <button 
        class="btn-page" 
        :disabled="page >= totalPages || loading" 
        @click="goToPage(totalPages)"
        title="末页"
      >
        »»
      </button>
    </div>
    
    <div class="pagination-size" v-if="showPageSize">
      <label>每页：</label>
      <select 
        v-model.number="localPageSize" 
        :disabled="loading"
        @change="onPageSizeChange"
        class="select-page-size"
      >
        <option v-for="size in pageSizeOptions" :key="size" :value="size">
          {{ size }}
        </option>
      </select>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  page: {
    type: Number,
    required: true,
    default: 1
  },
  pageSize: {
    type: Number,
    required: true,
    default: 10
  },
  total: {
    type: Number,
    required: true,
    default: 0
  },
  totalPages: {
    type: Number,
    required: true,
    default: 1
  },
  loading: {
    type: Boolean,
    default: false
  },
  showPageSize: {
    type: Boolean,
    default: true
  },
  pageSizeOptions: {
    type: Array,
    default: () => [5, 10, 20, 50, 100]
  },
  maxVisiblePages: {
    type: Number,
    default: 7
  }
})

const emit = defineEmits(['update:page', 'update:pageSize', 'change'])

const localPageSize = ref(props.pageSize)

// 可见页码计算
const visiblePages = computed(() => {
  const pages = []
  const maxVisible = props.maxVisiblePages
  let start = Math.max(1, props.page - Math.floor(maxVisible / 2))
  let end = Math.min(props.totalPages, start + maxVisible - 1)
  
  // 如果接近末尾，调整起始位置
  if (end - start < maxVisible - 1) {
    start = Math.max(1, end - maxVisible + 1)
  }
  
  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  return pages
})

function goToPage(newPage) {
  if (newPage < 1 || newPage > props.totalPages || props.loading) return
  emit('update:page', newPage)
  emit('change', { page: newPage, pageSize: props.pageSize })
}

function onPageSizeChange() {
  emit('update:pageSize', localPageSize.value)
  emit('update:page', 1) // 重置到第一页
  emit('change', { page: 1, pageSize: localPageSize.value })
}

// 同步外部 pageSize 变化
watch(() => props.pageSize, (newVal) => {
  localPageSize.value = newVal
})
</script>

<style scoped lang="scss">
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  flex-wrap: wrap;
}

.pagination-info {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: #666;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-page {
  min-width: 36px;
  height: 36px;
  padding: 0 8px;
  border: 1px solid rgba(0, 0, 0, 0.12);
  border-radius: 6px;
  background: #fff;
  color: #333;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
  
  &:hover:not(:disabled) {
    background: #f3f4f6;
    border-color: rgba(0, 0, 0, 0.2);
  }
  
  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  
  &.active {
    background: #2563eb;
    color: #fff;
    border-color: #2563eb;
    
    &:hover {
      background: #1d4ed8;
    }
  }
}

.page-numbers {
  display: flex;
  gap: 4px;
  margin: 0 4px;
}

.pagination-size {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  
  label {
    color: #666;
  }
}

.select-page-size {
  height: 36px;
  padding: 0 8px;
  border: 1px solid rgba(0, 0, 0, 0.12);
  border-radius: 6px;
  background: #fff;
  color: #333;
  font-size: 14px;
  cursor: pointer;
  
  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  
  &:hover:not(:disabled) {
    border-color: rgba(0, 0, 0, 0.2);
  }
}

@media (max-width: 768px) {
  .pagination {
    flex-direction: column;
    align-items: stretch;
  }
  
  .pagination-info {
    justify-content: center;
  }
  
  .pagination-controls {
    justify-content: center;
  }
  
  .pagination-size {
    justify-content: center;
  }
}
</style>

