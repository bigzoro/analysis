<template>
  <!-- 现代化页面头部 -->
  <section class="page-header">
    <div class="header-gradient">
      <div class="header-content">
        <!-- 面包屑导航 -->
        <nav class="breadcrumb-nav">
          <div class="breadcrumb">
            <span class="breadcrumb-item">
              <i class="icon-chart">📊</i>
              数据监控
            </span>
            <span class="breadcrumb-separator">/</span>
            <span class="breadcrumb-item active">
              <i class="icon-whale">🐋</i>
              大户监控
            </span>
          </div>
        </nav>

        <!-- 标题区域 -->
        <div class="title-section">
          <div class="title-content">
            <h1 class="page-title">
              大户 & 机构地址监控
            </h1>
            <p class="page-subtitle">
              实时监控区块链大户和机构的资金流动，支持多数据源智能聚合分析
            </p>
          </div>
          <div class="title-visual">
            <div class="floating-shapes">
              <div class="shape shape-1"></div>
              <div class="shape shape-2"></div>
              <div class="shape shape-3"></div>
            </div>
          </div>
        </div>

        <!-- 紧凑的控制面板 -->
        <div class="header-controls">
          <div class="control-row">
            <!-- 数据源选择 -->
            <div class="control-item">
              <label class="control-label">
                <i class="icon-data-source">💾</i>
                数据源
              </label>
              <div class="select-container">
                <select v-model="dataSource" @change="$emit('data-source-change', dataSource)" class="modern-select">
                  <option value="basic">📊 基本监控</option>
                  <option value="arkham">🔍 Arkham</option>
                  <option value="nansen">📈 Nansen</option>
                </select>
                <i class="select-arrow">▼</i>
              </div>
            </div>

            <!-- 实体选择 -->
            <div class="control-item">
              <label class="control-label">
                <i class="icon-entity">🏢</i>
                默认实体
              </label>
              <div class="select-container">
                <select v-model="entity" @change="$emit('entity-change', entity)" class="modern-select">
                  <option v-for="ent in entities" :key="ent" :value="ent">{{ ent }}</option>
                </select>
                <i class="select-arrow">▼</i>
              </div>
            </div>

            <!-- 快速操作按钮 -->
            <div class="control-actions">
              <button
                class="btn-primary btn-compact"
                @click="$emit('refresh-data')"
                :disabled="loading"
                :class="{ loading }"
                title="刷新所有监控地址的最新数据"
              >
                <i class="icon-refresh" :class="{ spinning: loading }">🔄</i>
                <span class="btn-text">{{ loading ? '刷新中' : '刷新数据' }}</span>
              </button>
              <button
                v-if="dataSource !== 'basic'"
                class="btn-secondary btn-compact"
                @click="$emit('sync-external')"
                :disabled="syncing"
                :class="{ loading: syncing }"
                title="从外部数据源同步最新数据"
              >
                <i class="icon-sync" :class="{ spinning: syncing }">⚡</i>
                <span class="btn-text">{{ syncing ? '同步中' : '外部同步' }}</span>
              </button>
              <button
                class="btn-outline btn-compact"
                @click="$emit('toggle-query-panel')"
                title="切换查询面板"
              >
                <i class="icon-query">{{ showQueryPanel ? '🔍' : '🔎' }}</i>
                <span class="btn-text">{{ showQueryPanel ? '隐藏查询' : '显示查询' }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

defineProps({
  dataSource: {
    type: String,
    default: 'basic'
  },
  entity: {
    type: String,
    default: 'binance'
  },
  entities: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  },
  syncing: {
    type: Boolean,
    default: false
  },
  showQueryPanel: {
    type: Boolean,
    default: true
  }
})

defineEmits(['data-source-change', 'entity-change', 'refresh-data', 'sync-external', 'toggle-query-panel'])
</script>

<style scoped>
/* 现代化页面头部样式 */
.page-header {
  margin-bottom: 2rem;
  overflow: hidden;
}

.header-gradient {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  border: 1px solid #475569;
  border-radius: 16px;
  position: relative;
}

.header-gradient::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(circle at 20% 80%, rgba(120, 119, 198, 0.3) 0%, transparent 50%),
              radial-gradient(circle at 80% 20%, rgba(255, 119, 198, 0.15) 0%, transparent 50%);
  border-radius: 16px;
  pointer-events: none;
}

.header-content {
  padding: 3rem 2rem;
  position: relative;
  z-index: 2;
}

/* 面包屑导航 */
.breadcrumb-nav {
  margin-bottom: 1.5rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.breadcrumb-item {
  color: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-weight: 500;
}

.breadcrumb-item.active {
  color: white;
  font-weight: 600;
}

.breadcrumb-separator {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.75rem;
}

/* 标题区域 */
.title-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2rem;
  gap: 2rem;
}

.title-content {
  flex: 1;
}

.page-title {
  font-size: 2.5rem;
  font-weight: 700;
  color: white;
  margin: 0 0 0.5rem 0;
  line-height: 1.2;
}

.page-subtitle {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.9);
  margin: 0;
  line-height: 1.6;
  max-width: 600px;
}

/* 浮动装饰元素 */
.title-visual {
  flex-shrink: 0;
  position: relative;
  width: 120px;
  height: 120px;
}

.floating-shapes {
  position: relative;
  width: 100%;
  height: 100%;
}

.shape {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.shape-1 {
  width: 40px;
  height: 40px;
  top: 20px;
  left: 30px;
}

.shape-2 {
  width: 25px;
  height: 25px;
  top: 60px;
  right: 20px;
}

.shape-3 {
  width: 15px;
  height: 15px;
  bottom: 30px;
  left: 50px;
}

/* 控制面板 */
.header-controls {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 16px;
  padding: 1.5rem;
}

.control-row {
  display: flex;
  align-items: flex-end;
  gap: 2rem;
  flex-wrap: wrap;
}

.control-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-width: 180px;
}

.control-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.select-container {
  position: relative;
}

.modern-select {
  width: 100%;
  padding: 0.75rem 1rem;
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 8px;
  color: white;
  font-size: 0.875rem;
  font-weight: 500;
  appearance: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modern-select:focus {
  outline: none;
  border-color: rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.25);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.1);
}

.modern-select option {
  background: white;
  color: #374151;
  padding: 0.5rem;
}

.select-arrow {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: rgba(255, 255, 255, 0.8);
  font-size: 0.75rem;
  pointer-events: none;
}

/* 操作按钮 */
.control-actions {
  display: flex;
  gap: 0.75rem;
  margin-left: auto;
}

.btn-compact {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.btn-primary {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-primary:hover:not(:disabled) {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.1);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.btn-secondary:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.5);
}

.btn-outline {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
}

.btn-outline:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.5);
}

.btn-compact:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-compact.loading {
  position: relative;
}

.btn-text {
  display: inline-block;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .header-content {
    padding: 2rem 1.5rem;
  }

  .page-title {
    font-size: 2rem;
  }

  .control-row {
    gap: 1.5rem;
  }

  .control-item {
    min-width: 160px;
  }
}

@media (max-width: 768px) {
  .title-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 1.5rem;
  }

  .title-visual {
    width: 80px;
    height: 80px;
  }

  .page-title {
    font-size: 1.75rem;
  }

  .control-row {
    flex-direction: column;
    align-items: stretch;
    gap: 1rem;
  }

  .control-item {
    min-width: auto;
  }

  .control-actions {
    margin-left: 0;
    justify-content: center;
  }

  .btn-compact {
    flex: 1;
    justify-content: center;
  }
}
</style>
