<template>
  <Teleport to="body">
    <div class="toast-container">
      <transition-group name="toast" tag="div">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="['toast-item', `toast-${toast.type}`]"
          @click="removeToast(toast.id)"
        >
          <div class="toast-content">
            <div class="toast-icon">{{ toast.icon }}</div>
            <div class="toast-text">
              <div class="toast-title">{{ toast.title }}</div>
              <div v-if="toast.message" class="toast-message">{{ toast.message }}</div>
            </div>
            <button class="toast-close" @click.stop="removeToast(toast.id)">
              ✕
            </button>
          </div>

          <!-- 进度条 -->
          <div class="toast-progress">
            <div
              class="progress-bar"
              :style="{ animationDuration: `${toast.duration}ms` }"
            ></div>
          </div>
        </div>
      </transition-group>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const toasts = ref([])
let toastId = 0

const addToast = (options) => {
  const id = ++toastId

  const toast = {
    id,
    type: options.type || 'info',
    title: options.title || '',
    message: options.message || '',
    duration: options.duration || 4000,
    icon: getIcon(options.type),
    ...options
  }

  toasts.value.push(toast)

  // 自动移除
  if (toast.duration > 0) {
    setTimeout(() => {
      removeToast(id)
    }, toast.duration)
  }

  return id
}

const removeToast = (id) => {
  const index = toasts.value.findIndex(t => t.id === id)
  if (index > -1) {
    toasts.value.splice(index, 1)
  }
}

const getIcon = (type) => {
  const icons = {
    success: '✅',
    error: '❌',
    warning: '⚠️',
    info: 'ℹ️'
  }
  return icons[type] || icons.info
}

// 便捷方法
const success = (title, message, options = {}) => {
  return addToast({ type: 'success', title, message, ...options })
}

const error = (title, message, options = {}) => {
  return addToast({ type: 'error', title, message, ...options })
}

const warning = (title, message, options = {}) => {
  return addToast({ type: 'warning', title, message, ...options })
}

const info = (title, message, options = {}) => {
  return addToast({ type: 'info', title, message, ...options })
}

// 暴露方法给父组件
defineExpose({
  add: addToast,
  remove: removeToast,
  success,
  error,
  warning,
  info
})

// 清理定时器
onUnmounted(() => {
  toasts.value = []
})
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: var(--space-4);
  right: var(--space-4);
  z-index: 10000;
  pointer-events: none;
}

.toast-item {
  pointer-events: auto;
  display: flex;
  flex-direction: column;
  margin-bottom: var(--space-3);
  min-width: 320px;
  max-width: 480px;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--border-light);
  overflow: hidden;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.toast-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-xl);
}

.toast-success {
  background: var(--success-50);
  border-color: var(--success-200);
}

.toast-error {
  background: var(--error-50);
  border-color: var(--error-200);
}

.toast-warning {
  background: var(--warning-50);
  border-color: var(--warning-200);
}

.toast-info {
  background: var(--primary-50);
  border-color: var(--primary-200);
}

.toast-content {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4);
}

.toast-icon {
  font-size: 1.25rem;
  flex-shrink: 0;
  margin-top: 2px;
}

.toast-text {
  flex: 1;
  min-width: 0;
}

.toast-title {
  font-weight: var(--font-semibold);
  font-size: var(--text-sm);
  color: var(--text-primary);
  margin-bottom: var(--space-1);
  line-height: 1.4;
}

.toast-message {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  line-height: 1.4;
}

.toast-close {
  flex-shrink: 0;
  background: none;
  border: none;
  font-size: var(--text-sm);
  color: var(--text-muted);
  cursor: pointer;
  padding: var(--space-1);
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
  margin-top: -2px;
}

.toast-close:hover {
  background: rgba(0, 0, 0, 0.1);
  color: var(--text-primary);
}

.toast-progress {
  height: 3px;
  background: rgba(0, 0, 0, 0.1);
  position: relative;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: currentColor;
  animation: progress linear;
  transform-origin: left;
}

.toast-success .progress-bar {
  background: var(--success-500);
}

.toast-error .progress-bar {
  background: var(--error-500);
}

.toast-warning .progress-bar {
  background: var(--warning-500);
}

.toast-info .progress-bar {
  background: var(--primary-500);
}

@keyframes progress {
  0% {
    transform: scaleX(1);
  }
  100% {
    transform: scaleX(0);
  }
}

/* 动画 */
.toast-enter-active,
.toast-leave-active {
  transition: all var(--transition-normal);
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
  max-height: 0;
  margin-bottom: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.toast-move {
  transition: transform var(--transition-normal);
}

/* 响应式设计 */
@media (max-width: 480px) {
  .toast-container {
    top: var(--space-2);
    right: var(--space-2);
    left: var(--space-2);
  }

  .toast-item {
    min-width: auto;
    max-width: none;
  }

  .toast-content {
    padding: var(--space-3);
    gap: var(--space-2);
  }

  .toast-icon {
    font-size: 1rem;
  }

  .toast-title {
    font-size: var(--text-xs);
  }

  .toast-message {
    font-size: 10px;
  }
}

/* 深色模式支持 */
@media (prefers-color-scheme: dark) {
  .toast-container {
    /* 深色模式下的样式 */
  }

  .toast-success {
    background: rgba(34, 197, 94, 0.1);
    border-color: rgba(34, 197, 94, 0.3);
  }

  .toast-error {
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.3);
  }

  .toast-warning {
    background: rgba(245, 158, 11, 0.1);
    border-color: rgba(245, 158, 11, 0.3);
  }

  .toast-info {
    background: rgba(59, 130, 246, 0.1);
    border-color: rgba(59, 130, 246, 0.3);
  }
}
</style>
