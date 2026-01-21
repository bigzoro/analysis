<template>
  <Teleport to="body">
    <transition name="fade">
      <div v-if="isLoading" class="global-loading-overlay">
        <div class="loading-container">
          <div class="loading-spinner">
            <div class="spinner-ring"></div>
            <div class="spinner-ring"></div>
            <div class="spinner-ring"></div>
            <div class="spinner-ring"></div>
          </div>

          <div class="loading-content">
            <div class="loading-icon">{{ icon }}</div>
            <div class="loading-text">{{ text }}</div>
            <div v-if="progress !== null" class="loading-progress">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: progress + '%' }"></div>
              </div>
              <div class="progress-text">{{ Math.round(progress) }}%</div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  text: {
    type: String,
    default: '加载中...'
  },
  icon: {
    type: String,
    default: '⏳'
  },
  progress: {
    type: Number,
    default: null
  }
})

const emit = defineEmits(['update:modelValue'])

const isLoading = ref(false)

watch(() => props.modelValue, (newVal) => {
  isLoading.value = newVal
})

watch(isLoading, (newVal) => {
  emit('update:modelValue', newVal)
})
</script>

<style scoped>
.global-loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-6);
  padding: var(--space-8);
  background: var(--bg-primary);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  border: 1px solid var(--border-light);
}

.loading-spinner {
  position: relative;
  width: 60px;
  height: 60px;
}

.spinner-ring {
  position: absolute;
  width: 100%;
  height: 100%;
  border: 3px solid transparent;
  border-top: 3px solid var(--primary-500);
  border-radius: 50%;
  animation: spin 1.2s cubic-bezier(0.5, 0, 0.5, 1) infinite;
}

.spinner-ring:nth-child(1) {
  animation-delay: -0.45s;
}

.spinner-ring:nth-child(2) {
  animation-delay: -0.3s;
}

.spinner-ring:nth-child(3) {
  animation-delay: -0.15s;
}

.spinner-ring:nth-child(4) {
  animation-delay: 0s;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-content {
  text-align: center;
  min-width: 200px;
}

.loading-icon {
  font-size: 2rem;
  margin-bottom: var(--space-3);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

.loading-text {
  font-size: var(--text-lg);
  font-weight: var(--font-medium);
  color: var(--text-primary);
  margin-bottom: var(--space-4);
}

.loading-progress {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-2xl);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-500), var(--primary-600));
  border-radius: var(--radius-2xl);
  transition: width 0.3s ease;
  position: relative;
}

.progress-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
  animation: shimmer 2s ease-in-out infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.progress-text {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
  min-width: 35px;
  text-align: right;
}

/* 响应式设计 */
@media (max-width: 480px) {
  .loading-container {
    padding: var(--space-6);
    gap: var(--space-4);
  }

  .loading-content {
    min-width: 150px;
  }

  .loading-icon {
    font-size: 1.5rem;
  }

  .loading-text {
    font-size: var(--text-base);
  }
}

/* 深色模式支持 */
@media (prefers-color-scheme: dark) {
  .global-loading-overlay {
    background: rgba(0, 0, 0, 0.8);
  }

  .loading-container {
    background: var(--bg-secondary);
    border-color: var(--border-dark);
  }
}
</style>
